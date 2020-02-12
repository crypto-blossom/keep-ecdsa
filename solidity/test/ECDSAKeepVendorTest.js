var Registry = artifacts.require('Registry')
var BondedECDSAKeepVendorStub = artifacts.require('BondedECDSAKeepVendorStub')
var ECDSAKeepFactoryVendorStub = artifacts.require('ECDSAKeepFactoryVendorStub')

contract("ECDSAKeepVendor", async accounts => {
    const address0 = "0x0000000000000000000000000000000000000000"
    const address1 = "0xF2D3Af2495E286C7820643B963FB9D34418c871d"
    const address2 = "0x4566716c07617c5854fe7dA9aE5a1219B19CCd27"

    let registry, keepVendor

    describe("registerFactory", async () => {
        beforeEach(async () => {
            registry = await Registry.new()
            keepVendor = await BondedECDSAKeepVendorStub.new()
            keepVendor.initialize(registry.address)
            await registry.setOperatorContractUpgrader(keepVendor.address, accounts[0])
            registry.approveOperatorContract(address0)
            registry.approveOperatorContract(address1)
            registry.approveOperatorContract(address2)
        })

        it("registers one factory address", async () => {
            let expectedResult = [address1]

            await keepVendor.registerFactory(address1)

            assertFactories(expectedResult)
        })

        it("registers factory with zero address", async () => {
            let expectedResult = [address0]

            await keepVendor.registerFactory(address0)

            assertFactories(expectedResult)
        })

        it("registers two factory addresses", async () => {
            let expectedResult = [address1, address2]

            await keepVendor.registerFactory(address1)
            await keepVendor.registerFactory(address2)

            assertFactories(expectedResult)
        })

        it("fails if address already exists", async () => {
            let expectedResult = [address1]

            await keepVendor.registerFactory(address1)

            try {
                await keepVendor.registerFactory(address1)
                assert(false, 'Test call did not error as expected')
            } catch (e) {
                assert.include(e.message, 'Factory address already registered')
            }

            assertFactories(expectedResult)
        })

        it("cannot be called by non-owner", async () => {
            let expectedResult = []

            try {
                await keepVendor.registerFactory(address1, { from: accounts[1] })
                assert(false, 'Test call did not error as expected')
            } catch (e) {
                assert.include(e.message, 'Caller is not operator contract upgrader')
            }

            assertFactories(expectedResult)
        })

        async function assertFactories(expectedFactories) {
            let result = await keepVendor.getFactories.call()
            assert.deepEqual(result, expectedFactories, "unexpected registered factories list")
        }
    })

    describe("selectFactory", async () => {
        before(async () => {
            keepVendor = await BondedECDSAKeepVendorStub.new()
            keepVendor.initialize(registry.address)
            await registry.setOperatorContractUpgrader(keepVendor.address, accounts[0])
            registry.approveOperatorContract(address1)
            registry.approveOperatorContract(address2)
        })

        it("returns last factory from the list", async () => {
            await keepVendor.registerFactory(address1)
            await keepVendor.registerFactory(address2)

            let expectedResult = address2

            let result = await keepVendor.selectFactoryPublic()

            assert.equal(result, expectedResult, "unexpected factory selected")
        })
    })
})
